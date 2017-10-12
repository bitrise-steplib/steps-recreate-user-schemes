package schemes

const xcodeprojGemfileContent = `source 'https://rubygems.org'
	
gem 'xcodeproj'
gem 'json'
`

const recreateUserSchemesRubyScriptContent = `require 'xcodeproj'
require 'json'

def contained_projects(project_or_workspace_pth)
  if File.extname(project_or_workspace_pth) == '.xcodeproj'
    [File.expand_path(project_or_workspace_pth)]
  else
    workspace = Xcodeproj::Workspace.new_from_xcworkspace(project_or_workspace_pth)
    workspace_dir = File.dirname(project_or_workspace_pth)
    project_paths = []
    workspace.file_references.each do |ref|
      pth = ref.path
      next unless File.extname(pth) == ".xcodeproj"
      next if pth.end_with?('Pods/Pods.xcodeproj')

      project_path = File.expand_path(pth, workspace_dir)
      project_paths << project_path
    end

    project_paths
  end
end

def read_user_schemes(project_path)
  user_name = ENV['USER']
  schemes = Dir[File.join(project_path, 'xcuserdata', user_name + '.xcuserdatad', 'xcschemes', '*.xcscheme')].map do |scheme|
    File.basename(scheme, '.xcscheme')
  end
  schemes
end

def read_share_schemes(project_path)
  schemes = Dir[File.join(project_path, 'xcshareddata', 'xcschemes', '*.xcscheme')].map do |scheme|
    File.basename(scheme, '.xcscheme')
  end
  schemes
end

def recreate_shared_schemes(project_path)  
  shared_schemes = read_share_schemes(project_path)  
  return unless shared_schemes.empty?

  user_schemes = read_user_schemes(project_path)  
  if user_schemes.empty?
    project = Xcodeproj::Project.open project_path
    project.recreate_user_schemes(true)
    user_schemes = read_user_schemes(project_path)    
  end  
  raise 'failed to recreate user schemes' if user_schemes.empty?

  user_schemes.each { |scheme| Xcodeproj::XCScheme.share_scheme(project_path, scheme) }

  shared_schemes = read_share_schemes(project_path)  
  raise 'failed to share user schemes' if shared_schemes.empty?
end

begin
  project_path = ENV['project_path']
  raise 'empty project_path' if project_path.empty?

  projects = contained_projects(project_path)
  projects.each { |project| recreate_shared_schemes(project) }
rescue => e
  error_message = e.to_s + "\n" + e.backtrace.join("\n")
  result = {
    error: error_message
  }
  result_json = result.to_json.to_s
  puts result_json
end
`
